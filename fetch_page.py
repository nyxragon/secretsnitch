import sys
import os
import hashlib
from selenium import webdriver
from selenium.webdriver.chrome.service import Service as ChromeService
from selenium.webdriver.chrome.options import Options

cache_dir = '.urlCache/'
extension = '.cache'

def fetch_page_source(url):
    chrome_options = Options()
    chrome_options.add_argument('--headless')
    chrome_options.add_argument('--no-sandbox')
    chrome_options.add_argument('--disable-dev-shm-usage')

    service = ChromeService(executable_path='/usr/bin/chromedriver')
    driver = webdriver.Chrome(service=service, options=chrome_options)

    try:
        driver.get(url)
        page_source = driver.page_source
        hash_object = hashlib.md5(url.encode())
        filename = hash_object.hexdigest()[:8] + extension
        with open(os.path.join(cache_dir, filename), 'w', encoding='utf-8') as file:
            file.write(url+"\n---\n"+page_source)
        print(f"Page source written to {os.path.join(cache_dir, filename)}")
        
    finally:
        driver.quit()

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python fetch_page.py <URL>")
        sys.exit(1)

    url = sys.argv[1]
    fetch_page_source(url)
